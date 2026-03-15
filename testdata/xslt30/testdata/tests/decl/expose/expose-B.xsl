<!-- simple use of xsl:expose to make specific variables and attribute sets final -->

<xsl:package
  name="http://www.w3.org/xslt30tests/expose-B"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/expose-B"
  xmlns:q="http://www.w3.org/xslt30tests/expose-B-private"
  exclude-result-prefixes="xs p">
  
  <xsl:expose visibility="private" component="variable" names="v1"/>
  <!-- Above declaration has no effect because the last match wins -->
  
  <xsl:expose visibility="final" component="attribute-set" names="a1 Q{http://www.w3.org/xslt30tests/expose-B}a2"/>
  <xsl:expose visibility="final" component="variable" names="v1 Q{http://www.w3.org/xslt30tests/expose-B}v2"/>
  
  <xsl:variable name="v1" select="0" visibility="public"/>
  <xsl:variable name="p:v2" select="0"/>
  <xsl:variable name="p:v3" select="0"/>
  
  <xsl:template name="t1">0</xsl:template>
  <xsl:template name="p:t2">0</xsl:template>
  
  <xsl:function name="p:f1"><xsl:sequence select="1"/></xsl:function>
  <xsl:function name="p:f2"><xsl:sequence select="2"/></xsl:function>
  <xsl:function name="q:f2"><xsl:sequence select="3"/></xsl:function>
  
  <xsl:attribute-set name="a1">
    <xsl:attribute name="A" select="0"/>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="a1">
    <xsl:attribute name="B" select="0"/>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="p:a2" visibility="public">
    <xsl:attribute name="A" select="0"/>
  </xsl:attribute-set>
  
  <xsl:mode name="m1"/>
  <xsl:mode name="p:m2"/>
  

</xsl:package>   