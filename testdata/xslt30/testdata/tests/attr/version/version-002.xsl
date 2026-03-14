<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="8.5">

  <!-- PURPOSE: Test that version attribute can be used on any XSLT element. 
       (This is an XSLT 2.0 feature)
       Test modified in Saxon 8.6 to use use-when rather than forwards-compatibility mode -->
       <?spec xslt#standard-attributes?>
  <xsl:template match="/">
    <out>
      <xsl:choose version="5.0">
        <xsl:when test="number(system-property('xsl:version')) >= 8.5"
                  use-when="number(system-property('xsl:version')) >= 8.5">
          <xsl:exciting-new-8.5-feature/>
        </xsl:when>
        <xsl:when test="true()">
          <xsl:text>This stylesheet requires XSLT 8.5 or higher</xsl:text>
        </xsl:when>
      </xsl:choose>
    </out>
  </xsl:template>
   
</xsl:stylesheet>
