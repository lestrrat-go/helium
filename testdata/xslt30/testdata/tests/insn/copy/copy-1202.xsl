<?xml version="1.0" encoding="ISO-8859-1" standalone="no"?>
<!DOCTYPE HTMLlat1 SYSTEM "htmllat1.dtd">
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#copy-of?>    
    <!-- Purpose: Test copy-of a string constant containing character entity -->

<xsl:output method="xml" encoding="UTF-8"/>
<!-- With this output encoding, should get two bytes (xC3,xA6) for the &aelig -->

<xsl:template match="/">
  <out>
    <xsl:copy-of select="'abcd&aelig;fgh'"/>
  </out>
</xsl:template>

</xsl:stylesheet>
