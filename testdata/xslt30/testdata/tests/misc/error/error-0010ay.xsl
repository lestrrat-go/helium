<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                     XSLT-defined element is used in a context where it is not permitted, if a
                        required attribute is omitted, or if the content of the
                     element does not correspond to the content that is allowed for the element.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
 	<xsl:output method="xml" version="2.0" encoding="UTF-8" indent="yes"/>
 
   <!-- variable declared as child of xsl:apply-imports -->
   <?spec xslt#errors?>  <?error XTSE0010?>
 
 
    <xsl:template match="/">
      <result>
        <xsl:apply-imports>
            <xsl:variable name="x" select="3"/>
            <xsl:with-param name="y" select="$x"/>
        </xsl:apply-imports>
      </result>
    </xsl:template>

</xsl:stylesheet>
