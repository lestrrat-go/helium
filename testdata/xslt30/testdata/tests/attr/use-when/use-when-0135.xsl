<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" xml:base="http://www.example.com/dir/">


  <!-- PURPOSE: test that xml:base affects the static base URI of a static expression. -->
  
  <xsl:variable name="inc" select="static-base-uri()" static="yes" xml:base="../sub"/>  
    
  <xsl:template name="main" use-when="$inc eq 'http://www.example.com/sub'">
    <ok/>
  </xsl:template>

</xsl:stylesheet>