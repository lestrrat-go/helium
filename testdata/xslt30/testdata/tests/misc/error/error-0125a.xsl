<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the value
                        of an [xsl:]default-collation attribute, after resolving
                        against the base URI, contains no URI that the implementation recognizes as
                        a collation URI.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                default-collation="mailto:unrecognized.collation@nowhere.com">

<?error XTSE0125?>


  <xsl:template name="main">
      <out>
         <xsl:apply-templates/>
      </out>
  </xsl:template>


</xsl:stylesheet>
