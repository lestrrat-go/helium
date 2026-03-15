<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                     XSLT-defined element is used in a context where it is not permitted, if a
                        required attribute is omitted, or if the content of the
                     element does not correspond to the content that is allowed for the element.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test validation when version matches supported version. -->
  <!-- This stylesheet should flag an error. -->
  <?spec xslt#errors?><?error XTSE0010?>
  
  <xsl:template match="/">
      <out>
         <xsl:choose>
            <xsl:when test="number(system-property('xsl:version')) &gt;= 8.0">
               <xsl:exciting-new-8.5-feature/>
            </xsl:when>
            <xsl:otherwise>
               <xsl:message>This stylesheet requires XSLT 1.0 or higher</xsl:message>
            </xsl:otherwise>
         </xsl:choose>
      </out>
  </xsl:template>
   
</xsl:stylesheet>
