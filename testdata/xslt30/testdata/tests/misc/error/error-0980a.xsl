<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if any undiscarded item in the atomized sequence supplied as
                     the value of the value attribute of xsl:number
                     cannot be converted to an integer, or if the resulting integer is less than 0
                     (zero). -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0980?>


  <xsl:template name="main">
      <out>
        <xsl:number value="-1"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
