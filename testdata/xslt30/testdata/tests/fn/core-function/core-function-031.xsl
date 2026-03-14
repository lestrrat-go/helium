<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-string-length?>
  <!-- Purpose: Test of 'string-length()' function with a variable reference 
       as the argument.-->

<xsl:variable name="variable1">should return the value 26</xsl:variable>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="string-length($variable1)"/>
 </out>
</xsl:template>
</xsl:stylesheet>