<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error for an element
                     from the XSLT namespace to have an attribute whose namespace is either null
                     (that is, an attribute with an unprefixed name) or the XSLT namespace, other
                     than attributes defined for the element in this document.-->
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
