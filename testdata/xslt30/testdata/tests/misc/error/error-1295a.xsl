<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the character
                     specified in the zero-digit attribute is not a digit or is a digit
                     that does not have the numeric value zero. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1295?>

   <xsl:decimal-format name="d" zero-digit="2"/>






  <xsl:template name="main">
      <xsl:variable name="e" as="element()">
         <e/>
      </xsl:variable>
      <xsl:variable name="d">
         <e/>
      </xsl:variable>    
      <out>
        <xsl:sequence select="format-number(12, '##0', 'd')"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
