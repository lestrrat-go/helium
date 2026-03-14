<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-string?>
  <!-- Purpose: Test of 'string()' function with 'string()' as an
       argument.-->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="string(string('Unchanged String'))"/>
 </out>
</xsl:template>
</xsl:stylesheet>