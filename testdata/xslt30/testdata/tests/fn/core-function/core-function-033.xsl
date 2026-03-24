<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-translate?>
  <!-- Purpose: Test of 'translate()' function with a variable reference 
       as the second argument.-->

<xsl:variable name="variable1">1234</xsl:variable>

<xsl:template match="doc">
 <out>
   <xsl:value-of select="translate('translate 1234 variable',$variable1,'with')"/>
 </out>
</xsl:template>
</xsl:stylesheet>