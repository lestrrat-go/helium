<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="8.5">

  <!-- PURPOSE: Test the basics of the XSLT version declaration. 
  Test modified in Saxon 8.6 to use use-when rather than forwards-compatibility mode -->
        <?spec xslt#standard-attributes?>
        <?spec xslt#forwards?>
  <xsl:template match="/">
    <out>
      <xsl:choose>
        <xsl:when test="number(system-property('xsl:version')) >= 8.5"
                    use-when="number(system-property('xsl:version')) >= 8.5">
          <xsl:exciting-new-8.5-feature/>
        </xsl:when>
        <xsl:when test="true()">
          <xsl:message>This stylesheet requires XSLT 8.5 or higher</xsl:message>
        </xsl:when>
      </xsl:choose>
    </out>
  </xsl:template>
   
</xsl:stylesheet>
