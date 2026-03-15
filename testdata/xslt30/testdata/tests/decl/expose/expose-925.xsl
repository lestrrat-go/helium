<!-- xsl:expose - error, component not in package -->

<xsl:package
  name="http://www.w3.org/xslt30tests/package-902"  
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
  <xsl:function name="p:f2"><xsl:sequence select="0"/></xsl:function>
  
  <xsl:attribute-set name="a1">
    <xsl:attribute name="A" select="0"/>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="a1">
    <xsl:attribute name="B" select="0"/>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="p:a2">
    <xsl:attribute name="A" select="0"/>
  </xsl:attribute-set>
  
  <xsl:mode name="m1"/>
  <xsl:mode name="p:m2"/>
  

    <xsl:template name="main" visibility="public">
      <ok/>
    </xsl:template>
  
  <xsl:template name="xsl:initial-template" visibility="public">
    <wrong/>
  </xsl:template>
    
    <xsl:expose visibility="public" component="template" names="t1#0"/>

</xsl:package>   