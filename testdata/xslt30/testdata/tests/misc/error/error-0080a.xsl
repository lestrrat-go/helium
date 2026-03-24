<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error to use a reserved namespace in the name of a
                        named template, a mode, an attribute
                        set, a key, a decimal-format, a variable or parameter, a stylesheet
                        function, a named output
                        definition, or a character
                        map.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?error XTSE0080?>


  <xsl:template name="xsl:main">
      <xsl:param name="y" required="yes"/>
      <out>
         <xsl:apply-templates select="$y"/>
      </out>
  </xsl:template>

</xsl:stylesheet>
