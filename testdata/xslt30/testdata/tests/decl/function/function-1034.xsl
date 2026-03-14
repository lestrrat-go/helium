<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:x = "http://xxx.com/"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs x"
  expand-text="yes">

<!-- cached functions. We don't actually check that the function results are cached, but we try to ensure that if
   caching is done, then it's done correctly. The xsl:message calls are there to help establish whether the result
   was computed or obtained from cache, but they are not part of the definitive test results  -->

<?spec xslt#stylesheet-functions?>

<xsl:function name="x:int-int" cache="yes">
  <xsl:param name="a" as="xs:integer"/>
  <xsl:param name="b" as="xs:integer"/>
  <xsl:message>Evaluating xs:int-int({$a},{$b})</xsl:message>
  <xsl:sequence select="2*$a + $b"/>
</xsl:function>
  
  <xsl:function name="x:str-str" cache="yes">
    <xsl:param name="a" as="xs:string"/>
    <xsl:param name="b" as="xs:string"/>
    <xsl:message>Evaluating xs:str-str({$a},{$b})</xsl:message>
    <xsl:sequence select="2*string-length($a) + string-length($b)"/>
  </xsl:function>
  
  <xsl:function name="x:qn-qn" cache="yes">
    <xsl:param name="a" as="xs:QName"/>
    <xsl:param name="b" as="xs:QName"/>
    <xsl:message>Evaluating xs:qn-qn({namespace-uri-from-QName($a)}:{$a},{namespace-uri-from-QName($b)}:{$b})</xsl:message>
    <xsl:sequence select="$a eq $b"/>
  </xsl:function>
  
  <xsl:function name="x:node-node" cache="yes">
    <xsl:param name="a" as="node()"/>
    <xsl:param name="b" as="node()"/>
    <xsl:message>Evaluating xs:node-node({namespace-uri($a)}:{local-name($a)},{namespace-uri($b)}:{local-name($b)}))</xsl:message>
    <xsl:sequence select="$a is $b"/>
  </xsl:function>
  
  <xsl:function name="x:map-map" cache="yes">
    <xsl:param name="a" as="map(*)"/>
    <xsl:param name="b" as="map(*)"/>
    <xsl:message>Evaluating xs:map-map({$a?id},{$b?id}))</xsl:message>
    <xsl:sequence select="$a?value || '~' || $b?value"/>
  </xsl:function>
  
  <xsl:function name="x:fn-fn" cache="yes">
    <xsl:param name="a" as="function(*)"/>
    <xsl:param name="b" as="function(*)"/>
    <xsl:message>Evaluating xs:fn-fn(...))</xsl:message>
    <xsl:sequence select="$a($b('x', 'y'), $b('p', 'q'))"/>
  </xsl:function>
  
  <xsl:template name="xsl:initial-template">
    <out>
      <int-int>
        <xsl:message>5 calls on x:int-int()</xsl:message>
        <a>{x:int-int(2, 0)}</a>
        <b>{x:int-int(0, 2)}</b>
        <c>{x:int-int(2, 0)}</c>
        <d>{x:int-int(-2, 1)}</d>
        <e>{x:int-int(2, 0)}</e>
      </int-int>
      <str-str>
        <xsl:message>5 calls on x:str-str()</xsl:message>
        <a>{x:str-str('alpha', 'beta')}</a>
        <b>{x:str-str('', 'alphabeta')}</b>
        <c>{x:str-str('alphabeta', '')}</c>
        <d>{x:str-str('alpha', 'beta')}</d>
      </str-str>
      <qn-qn>
        <xsl:variable name="data">
          <x:alpha id='1' xmlns:x="http://ns.one/"/>
          <x:alpha id='2' xmlns:x="http://ns.two/"/>
        </xsl:variable>
        <xsl:message>3 calls on x:qn-qn()</xsl:message>
        <a>{x:qn-qn($data/*:alpha[1]/node-name(), $data/*:alpha[1]/node-name())}</a>
        <b>{x:qn-qn($data/*:alpha[1]/node-name(), $data/*:alpha[2]/node-name())}</b>
        <c>{x:qn-qn($data/*:alpha[1]/node-name(), $data/*:alpha[1]/node-name())}</c>
      </qn-qn>
      <node-node>
        <xsl:variable name="data">
          <alpha/>
          <alpha/>
        </xsl:variable>
        <xsl:message>3 calls on x:node-node()</xsl:message>
        <a>{x:node-node($data/alpha[1], $data/alpha[1])}</a>
        <b>{x:node-node($data/alpha[1], $data/alpha[2])}</b>
        <c>{x:node-node($data/alpha[1], $data/alpha[1])}</c>
      </node-node>
      <map-map>
        <xsl:message>3 calls on x:map-map()</xsl:message>
        <xsl:variable name="m" select="map{'id':3, 'value':30}"/>
        <xsl:variable name="n" select="map{'id':3, 'value':30}"/>
        <xsl:variable name="o" select="map{'id':3, 'value':50}"/>
        <a>{x:map-map($m, $m)}</a>
        <b>{x:map-map($m, $n)}</b>
        <c>{x:map-map($m, $o)}</c>
        <d>{x:map-map($m, $m)}</d>
      </map-map>
      <fn-fn>
        <xsl:message>3 calls on x:fn-fn()</xsl:message>
        <xsl:variable name="m" select="concat#2"/>
        <xsl:variable name="n" select="function($x,$y){$x||$y}"/>
        <xsl:variable name="o" select="concat(?, ?, '')"/>
        <a>{x:fn-fn($m, $m)}</a>
        <b>{x:fn-fn($m, $n)}</b>
        <c>{x:fn-fn($m, $o)}</c>
        <d>{x:fn-fn($m, $m)}</d>
      </fn-fn>
    </out>
  </xsl:template>

</xsl:stylesheet>
