<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- shadow attribute (static AVT) on @use-when -->
   
   <xsl:variable name="V" static="yes" select="3"/>
   
   <xsl:template name="main" _use-when="$V = {'3'}">
      <out>3</out>
   </xsl:template>
   
   <xsl:template name="main" _use-when="$V = {'4'}">
      <out>4</out>
   </xsl:template>
   
</xsl:transform>
