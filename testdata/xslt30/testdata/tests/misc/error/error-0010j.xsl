<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                     XSLT-defined element is used in a context where it is not permitted, if a
                        required attribute is omitted, or if the content of the
                     element does not correspond to the content that is allowed for the element.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:choose  -->

  <xsl:template match="doc" name="this">
  <?spec xslt#errors?><?error XTSE0010?>
      <out>
         <xsl:choose>
            <xsl:when test="2=2">butter</xsl:when>
            <xsl:when match="2=4">cheese</xsl:when>
         </xsl:choose>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>
  

</xsl:stylesheet>
