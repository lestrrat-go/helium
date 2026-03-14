<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an unescaped
                     left curly bracket appears in a fixed part of a content value template
                     without a matching right curly bracket.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                version="2.0"
                exclude-result-prefixes="xs">

<!-- PURPOSE:  unclosed expression in AVT  -->
<?spec xslt#errors?><?error XTSE0350?>

   <xsl:template match="/">
      <A b="x{3}y{4"/>
   </xsl:template>

</xsl:stylesheet>
