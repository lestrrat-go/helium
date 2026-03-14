<?xml version="1.0" encoding="UTF-8"?>
<!--It 
               is a static error if 
               within any package
               the same NameTest
            appears in both an xsl:strip-space and an xsl:preserve-space
               declaration if both have the same import
               precedence. Two NameTests are considered the same if they match the same set of names
               (which can be determined by comparing them after expanding namespace prefixes to URIs).-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<?error XTRE0270?>
<!-- Either signals the error, or reports two (whitespace) text nodes -->

   <xsl:strip-space elements="doc"/>

  <xsl:template name="main">
      <out>
         <xsl:apply-templates/>
      </out>
  </xsl:template>
  
   <xsl:preserve-space elements="doc"/>

   <xsl:template match="doc">
      <xsl:value-of select="count(text())"/>
   </xsl:template>


</xsl:stylesheet>
