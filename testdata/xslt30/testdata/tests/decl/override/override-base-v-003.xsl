<xsl:package 
  name="http://www.w3.org/xslt30tests/override-base-v-003"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  
  <xsl:variable name="var" as="xs:int" select="xs:int(2)" visibility="public"/>
  
  <xsl:param name="req" as="xs:NCName" required="yes"/>
  
  <xsl:param name="opt" as="xs:date" select="current-date()" required="no"/>
  
  

 
  
</xsl:package>   