<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a character
                     map references itself, directly or indirectly, via a name in the
                        use-character-maps attribute.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1600?>

   <xsl:character-map name="one" use-character-maps="two">
      <xsl:output-character character="a" string="aa"/>
   </xsl:character-map>

   <xsl:character-map name="two" use-character-maps="three">
      <xsl:output-character character="b" string="bb"/>
   </xsl:character-map>

   <xsl:character-map name="three" use-character-maps="one">
      <xsl:output-character character="c" string="cc"/>
   </xsl:character-map>


  <xsl:template name="main">
      <out>
         <e/>
      </out>
  </xsl:template>
  




</xsl:stylesheet>
