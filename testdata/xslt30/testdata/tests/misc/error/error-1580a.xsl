<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the stylesheet contains two or more character maps
                     with the same name and the same import
                        precedence, unless it also contains another character map with the
                     same name and higher import precedence.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1580?>

   <xsl:character-map name="one">
      <xsl:output-character character="a" string="aa"/>
   </xsl:character-map>

   <xsl:character-map name="one">
      <xsl:output-character character="b" string="bb"/>
   </xsl:character-map>


  <xsl:template name="main">
      <out>
         <e/>
      </out>
  </xsl:template>
  




</xsl:stylesheet>
