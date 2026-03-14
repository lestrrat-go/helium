<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a name in the
                        use-character-maps attribute of the xsl:output
                     or xsl:character-map elements does not match the
                        name attribute of any xsl:character-map in the
                        stylesheet.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1590?>

   <xsl:character-map name="one" use-character-maps="two">
      <xsl:output-character character="a" string="aa"/>
   </xsl:character-map>

   <xsl:character-map name="three">
      <xsl:output-character character="b" string="bb"/>
   </xsl:character-map>


  <xsl:template name="main">
      <out>
         <e/>
      </out>
  </xsl:template>
  




</xsl:stylesheet>
