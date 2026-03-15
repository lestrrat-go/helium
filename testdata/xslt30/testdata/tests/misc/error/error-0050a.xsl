<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the stylesheet that is invoked declares a visible
                        stylesheet parameter with
                        required="yes" and no value for this parameter is supplied
                     during the invocation of the stylesheet. A stylesheet parameter is visible if
                     it is not masked by another global variable or parameter with the same name and
                     higher import precedence.
                  If the parameter is a static parameter then
                  the value must be supplied prior to the static analysis phase.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?error XTDE0050?>

   <xsl:param name="x" required="yes"/>

  <xsl:template name="main">
      <out>
         <xsl:apply-templates select="$x"/>
      </out>
  </xsl:template>

</xsl:stylesheet>
