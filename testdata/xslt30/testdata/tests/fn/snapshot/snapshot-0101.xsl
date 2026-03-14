<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs"
  expand-text="yes"
  version="3.0">

  <xsl:template name="a">
    <xsl:variable name="v" select="snapshot(/)"/>
    <xsl:choose>
      <xsl:when test="deep-equal(/, $v)">
        <ok/>
      </xsl:when>
      <xsl:otherwise>
        <xsl:sequence select="$v"/>
      </xsl:otherwise>
    </xsl:choose>
  </xsl:template>

  <xsl:template name="b">
    <xsl:variable name="v" select="snapshot(/*)"/>
    <xsl:choose>
      <xsl:when test="deep-equal(/*, $v)">
        <ok doc="{exists($v/..)}"/>
      </xsl:when>
      <xsl:otherwise>
        <xsl:sequence select="$v"/>
      </xsl:otherwise>
    </xsl:choose>
  </xsl:template>

  <xsl:template name="c">
    <xsl:sequence select="snapshot(//*:identifier)/self::*:identifier/ancestor::node()[last()]"/>
  </xsl:template>

  <xsl:template name="d">
    <xsl:variable name="result" select="snapshot((//*:identifier, 1, 'Fred', name#1))"/>
    <out>
      <xsl:value-of select="$result[1] instance of element(*) and 
          $result[2] instance of xs:integer and 
          $result[3] instance of xs:string and
          $result[4] instance of function(*)"/>
    </out>
  </xsl:template>
  
  <xsl:template name="e">
    <xsl:variable name="result" select="snapshot(//@srsName)"/>
    <out>
      <xsl:message select="$result"/>
      <a>{count($result) eq 1}</a>
      <b>{$result[1] instance of attribute(srsName)}</b>
      <c>{exists($result[1]/parent::Q{http://www.opengis.net/gml/3.2}Envelope)}</c>
      <d>{empty($result[1]/child::node())}</d>
    </out>
  </xsl:template>
  
  <xsl:template name="f">
    <xsl:variable name="result" select="snapshot(//*:description/namespace::gml)"/>
    <out>
      <a>{count($result) eq 1}</a>
      <b>{$result[1] instance of namespace-node()}</b>
      <c>{local-name($result[1]) eq 'gml'}</c>
      <d>{string($result[1]) eq 'http://www.opengis.net/gml/3.2'}</d>
      <e>{exists($result[1]/parent::Q{http://www.opengis.net/gml/3.2}description)}</e>
      <f>{empty($result[1]/child::node())}</f>
    </out>
  </xsl:template>


</xsl:stylesheet>
