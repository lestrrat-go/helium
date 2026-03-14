<?xml version="1.0" encoding="UTF-8"?>
<!--The value of the method attribute on
                        xsl:output
                  
                  must (if present) be a valid EQName. If it is a lexical QName with no
                  a prefix, then it identifies a method specified in  and must be one of
                     xml, html, xhtml, or
                  text.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1570?>

   <xsl:output name="a" method="your::xml"/>


  <xsl:template name="main">
      <out>
         <e/>
      </out>
  </xsl:template>
  




</xsl:stylesheet>
