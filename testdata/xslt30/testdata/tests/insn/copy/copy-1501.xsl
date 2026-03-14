<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#copy-of?>    
  <!-- Purpose: Test for copy-of text nodes including CDATA. -->
  
<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:copy-of select="doc"/>
  </out>
</xsl:template>

</xsl:stylesheet>
