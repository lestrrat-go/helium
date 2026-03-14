<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the invocation of the stylesheet specifies a template name that does not match the
                        expanded QName of a named
                     template defined in the stylesheet.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?error XTDE0040?>

   <xsl:param name="x" select="'bananas'"/>

  <xsl:template name="not-main">
      <out>
         <xsl:apply-templates/>
      </out>
  </xsl:template>

</xsl:stylesheet>
