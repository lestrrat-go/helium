<!-- base stylesheet for testing variations of xsl:accept -->

<xsl:package
  id="accept-B"
  name="http://www.w3.org/xslt30tests/accept-B"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/accept-B"
  xmlns:q="http://www.w3.org/xslt30tests/accept-B-private"
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
  
  <xsl:expose component="function" names="q:f3#0" visibility="private"/>
  
  <xsl:variable name="v1" select="11"/>
  <xsl:variable name="p:v2" select="22"/>
  
  <xsl:template name="t1">0</xsl:template>
  <xsl:template name="p:t2">0</xsl:template>
  
  <xsl:function name="p:f1"><xsl:sequence select="11"/></xsl:function>
  <xsl:function name="p:f2"><xsl:sequence select="22"/></xsl:function>
  <xsl:function name="q:f2"><xsl:sequence select="33"/></xsl:function>
  <xsl:function name="q:f3"><xsl:sequence select="44"/></xsl:function>
  
  <xsl:attribute-set name="a1">
    <xsl:attribute name="A" select="'AA'"/>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="a1">
    <xsl:attribute name="B" select="'BB'"/>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="p:a2">
    <xsl:attribute name="A" select="1000"/>
  </xsl:attribute-set>
  
  <xsl:mode name="m1"/>
  <xsl:mode name="p:m2"/>
  

    <xsl:template name="main">
      <ok/>
    </xsl:template>
    


</xsl:package>   