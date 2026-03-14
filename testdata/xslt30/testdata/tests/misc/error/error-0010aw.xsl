<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                     XSLT-defined element is used in a context where it is not permitted, if a
                        required attribute is omitted, or if the content of the
                     element does not correspond to the content that is allowed for the element.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
 	<xsl:output method="xml" version="2.0" encoding="UTF-8" indent="yes"/>
 
   <!-- xsl:stylesheet not at top level -->
   <?spec xslt#errors?>  <?error XTSE0010?>
 
    <xsl:template match="/">
      <result>
        <xsl:attribute name="test_attr">
            <xsl:stylesheet version="2.0">xsl:stylesheet</xsl:stylesheet>
        </xsl:attribute>
      </result>
    </xsl:template>

</xsl:stylesheet>
