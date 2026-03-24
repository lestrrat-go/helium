<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">


  <!-- PURPOSE: test use-when="available-environment-variables()". -->

  <xsl:variable name="user" select="environment-variable('USER')"/>
  
  <xsl:mode on-no-match="shallow-copy"/>

  <xsl:template match="para" use-when="available-environment-variables() = 'USER' or empty(environment-variable('USER'))" priority="2">
    <ok/>
  </xsl:template>
  
  <xsl:template match="para" priority="1">
    <wrong/>
  </xsl:template>
  
  <xsl:strip-space elements="*"/>

</xsl:stylesheet>
