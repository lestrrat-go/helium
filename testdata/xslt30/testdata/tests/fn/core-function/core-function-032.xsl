<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-translate?>
  <!-- Purpose: Test of 'translate()' function with a variable reference 
       as the first argument.-->

<xsl:variable name="variable1">translate 1234 variable</xsl:variable>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="translate($variable1,'1234','with')"/>
 </out>
</xsl:template>
</xsl:stylesheet>