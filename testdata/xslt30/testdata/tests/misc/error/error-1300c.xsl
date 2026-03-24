<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if, for any named
                     or unnamed decimal format, the variables representing characters used in a
                        picture string do not each have
                     distinct values. These variables are decimal-separator-sign,
                        grouping-sign, percent-sign,
                     per-mille-sign, digit-zero-sign, digit-sign,
                     and pattern-separator-sign.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1300?>

   <xsl:decimal-format name="d" pattern-separator="'" decimal-separator="'"/>

  <xsl:template name="main">
      <out>
        <xsl:sequence select="format-number(12, '##0', 'd')"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
