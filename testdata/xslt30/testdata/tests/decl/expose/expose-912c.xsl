<!-- xsl:expose - error, cannot increase visibility -->

<xsl:package
  name="http://www.w3.org/xslt30tests/package-912c"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/use-package-base-001"
  exclude-result-prefixes="xs p">
  
  <xsl:variable name="v1" select="0"/>
  <xsl:variable name="p:v2" select="0"/>
  
  <xsl:template name="t1">0</xsl:template>
  <xsl:template name="p:t2">0</xsl:template>
  
  <xsl:function name="p:f1"><xsl:sequence select="0"/></xsl:function>
  <xsl:function name="p:f1"><xsl:param name="x"/><xsl:sequence select="0"/></xsl:function>
  <xsl:function name="p:f1"><xsl:param name="x"/><xsl:param name="y"/><xsl:sequence select="0"/></xsl:function>
  <xsl:function name="p:f2"><xsl:sequence select="0"/></xsl:function>
  
  <xsl:attribute-set name="a1">
    <xsl:attribute name="A" select="0"/>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="a1">
    <xsl:attribute name="B" select="0"/>
  </xsl:attribute-set>
  
  <xsl:param name="p"/>
  
  <xsl:mode/>
  <xsl:mode name="m1"/>
  <xsl:mode name="p:m2"/>
  
  <xsl:accumulator name="acc" initial-value="0">
    <xsl:accumulator-rule match="*" select="$value+1"/>
  </xsl:accumulator>  
  

    <xsl:template name="main" visibility="public">
      <ok/>
    </xsl:template>
    
    <xsl:expose visibility="public" component="template" names="t1 p:t2"/>
    <xsl:expose visibility="final" component="variable" names="v1"/>
    <xsl:expose visibility="public" component="variable" names="p:v2"/>
    <xsl:expose visibility="private" component="function" names="p:f2#0"/>
    <xsl:expose visibility="abstract" component="function" names="p:f1#0"/>
    
    <xsl:expose visibility="final" component="attribute-set" names="a1 *:a2"/>
    <xsl:expose visibility="public" component="mode" names="*:m2"/>
    
    <xsl:expose visibility="public" component="variable" names="*"/>

</xsl:package>   