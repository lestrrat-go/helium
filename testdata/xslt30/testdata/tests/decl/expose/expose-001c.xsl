<!-- syntax variations of xsl:expose, error case, initial template not public -->

<xsl:package
  name="http://www.w3.org/xslt30tests/expose-001"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/expose-001"
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
  

    <xsl:template name="main">
      <ok/>
    </xsl:template>
    
    <xsl:expose visibility="private" component="template" names="main"/>
    <xsl:expose visibility="public" component="template" names="t1 p:t2"/>
    <xsl:expose visibility="final" component="variable" names="v1 p:v2"/>
    <xsl:expose visibility="private" component="function" names="p:f1#0 p:f2#0"/>
    
    <xsl:expose visibility="public" component="attribute-set" names="p:* a1"/>
    <xsl:expose visibility="public" component="mode" names="*:m2 m1"/>
    
    <xsl:expose visibility="public" component="variable" names="*"/>

</xsl:package>   