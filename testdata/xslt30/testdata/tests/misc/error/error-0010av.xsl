<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                     XSLT-defined element is used in a context where it is not permitted, if a
                        required attribute is omitted, or if the content of the
                     element does not correspond to the content that is allowed for the element.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <!-- test for disabling output escaping in nested xsl:text -->
  
  <!-- now an error, XSLT 2.0 no longer allows nested xsl:text -->
  <?spec xslt#errors?><?error XTSE0010?>
  
  
  <xsl:output method="xml"/>
  <xsl:template match="/">
      <HTML>
         <BODY>
            <xsl:text>
               <xsl:text disable-output-escaping="no">[&amp;]</xsl:text>
               <xsl:text disable-output-escaping="yes">[&amp;nbsp;]</xsl:text>
               <xsl:text>[&amp;]</xsl:text>
            </xsl:text>   
            <xsl:text disable-output-escaping="yes">
               <xsl:text disable-output-escaping="no">[&amp;]</xsl:text>
               <xsl:text disable-output-escaping="yes">[&amp;nbsp;]</xsl:text>
               <xsl:text>[&amp;]</xsl:text>
            </xsl:text>                     
         </BODY>
      </HTML>
  </xsl:template>
</xsl:stylesheet>
