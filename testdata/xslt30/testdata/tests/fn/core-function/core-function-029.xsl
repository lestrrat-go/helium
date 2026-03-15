<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-substring?>
  <!-- Purpose: Test of 'substring()' function with a variable reference 
       as the first argument.  Third argument is missing.-->

<xsl:variable name="variable1">cut thissubstring with variable</xsl:variable>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="substring($variable1,9)"/>
 </out>
</xsl:template>
</xsl:stylesheet>