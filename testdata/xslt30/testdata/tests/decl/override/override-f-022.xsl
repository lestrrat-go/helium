<xsl:package 
  name="http://www.w3.org/xslt30tests/override-f-022"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-001"
  exclude-result-prefixes="xs p">
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-base-001"
    package-version="1.0.0"/>

  

  <xsl:function name="p:f" as="xs:string" visibility="public">
    <xsl:param name="in" as="xs:string"/>
    <xsl:param name="count" as="xs:integer"/>
    <xsl:variable name="x" as="element()">
      <x/>
    </xsl:variable>  
    <xsl:sequence select="generate-id($x)"/>
  </xsl:function>
 
  
    <!-- Call public function defined in used package -->
   
    <xsl:template name="main" visibility="public">
      <xsl:choose>
        <xsl:when test="p:f('5', 5) = 'xxxxx'"><ok/></xsl:when>
        <xsl:otherwise><wrong value="p:f('5', 5)"/></xsl:otherwise>
      </xsl:choose>
    </xsl:template>
    
 
  
</xsl:package>   