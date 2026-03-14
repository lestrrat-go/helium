<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if both the
                        [xsl:]type and [xsl:]validation attributes are
                     present on the xsl:element, xsl:attribute,
                        xsl:copy, xsl:copy-of,
                        xsl:document, or xsl:result-document
                     instructions, or on a literal result
                        element.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1505?>


  <xsl:template name="main">
    
      <out> 
         <xsl:element name="e" validation="preserve" type="xs:anyType"/>      
      </out>
  </xsl:template>
  




</xsl:stylesheet>
