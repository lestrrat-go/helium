<root xsl:version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
      <!-- test reporting of dynamic errors in a simplified stylesheet -->
      <?spec xslt#errors?><?error FODC0005?>
      <xsl:copy-of select="document('note.xml')/*"/>
</root>
