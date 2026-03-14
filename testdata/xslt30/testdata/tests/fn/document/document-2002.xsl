<root xsl:version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
   <?spec xslt#document?>
      <!-- test that a simplified stylesheet has the correct base URI -->
      <xsl:copy-of select="document('doc14.xml')/*"/>
</root>
