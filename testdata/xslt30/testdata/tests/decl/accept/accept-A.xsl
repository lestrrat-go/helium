<!-- base stylesheet for testing variations of xsl:accept -->

<xsl:package
  id="accept-A"
  name="http://www.w3.org/xslt30tests/accept-A"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/accept-A"
  xmlns:q="http://www.w3.org/xslt30tests/accept-A-private"
  exclude-result-prefixes="xs p">
  
  <xsl:expose component="variable" names="*" visibility="public"/>
  <xsl:expose component="template" names="*" visibility="public"/>
  <xsl:expose component="function" names="*" visibility="public"/>
  <xsl:expose component="attribute-set" names="*" visibility="public"/>
  <xsl:expose component="mode" names="*" visibility="public"/>
  
  <xsl:expose component="variable" names="p:*" visibility="final"/>
  <xsl:expose component="template" names="p:*" visibility="final"/>
  <xsl:expose component="function" names="p:*" visibility="final"/>
  <xsl:expose component="attribute-set" names="p:*" visibility="final"/>
  <xsl:expose component="mode" names="p:*" visibility="final"/>
  
  <xsl:expose component="*" names="q:*" visibility="private"/>
  
  <xsl:variable name="v1" select="1"/>
  <xsl:variable name="p:v2" select="2"/>
  
  <xsl:template name="t1">0</xsl:template>
  <xsl:template name="p:t2">0</xsl:template>
  
  <xsl:function name="p:f1"><xsl:sequence select="1"/></xsl:function>
  <xsl:function name="p:f2"><xsl:sequence select="2"/></xsl:function>
  <xsl:function name="q:f2"><xsl:sequence select="3"/></xsl:function>
  
  <xsl:attribute-set name="a1">
    <xsl:attribute name="A" select="'A'"/>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="a1">
    <xsl:attribute name="B" select="'B'"/>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="p:a2">
    <xsl:attribute name="A" select="0"/>
  </xsl:attribute-set>
  
  <xsl:mode name="m1"/>
  <xsl:mode name="p:m2"/>
  

    <xsl:template name="main" visibility="private">
      <ok/>
    </xsl:template>
    


</xsl:package>   