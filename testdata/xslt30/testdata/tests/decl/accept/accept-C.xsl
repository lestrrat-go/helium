<!-- base stylesheet for testing variations of xsl:accept -->

<xsl:package id="accept-C" name="http://www.w3.org/xslt30tests/accept-C" package-version="1.0.0"
  version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:C="http://www.w3.org/xslt30/tests/accept"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs C">


  <xsl:variable name="v1" as="xs:integer" visibility="abstract"/>
  
  <xsl:variable name="v1-proxy" as="xs:integer" visibility="public" select="$v1"/>

  <xsl:template name="t1" as="xs:integer" visibility="abstract">
    <xsl:param name="p1" as="xs:string"/>
  </xsl:template>
  
  <xsl:template name="t1-proxy" as="xs:integer" visibility="public">
    <xsl:param name="p1" as="xs:string"/>
    <xsl:call-template name="t1">
      <xsl:with-param name="p1" select="$p1"/>
    </xsl:call-template>
  </xsl:template>

  <xsl:attribute-set name="a1" visibility="abstract"/>
  
  <xsl:attribute-set name="a1-proxy" use-attribute-sets="a1" visibility="public"/>

  <xsl:function name="C:f1" as="xs:integer" visibility="abstract">
    <xsl:param name="p1" as="xs:string"/>
  </xsl:function>
  
  <xsl:function name="C:f1-proxy" as="xs:integer" visibility="public">
    <xsl:param name="p1" as="xs:string"/>
    <xsl:sequence select="C:f1($p1) + 1"/>
  </xsl:function>
  
  


</xsl:package>
