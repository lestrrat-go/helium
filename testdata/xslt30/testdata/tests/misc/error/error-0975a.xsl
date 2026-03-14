<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the
                     value attribute of xsl:number is present unless
                  the select, level, count, and
                     from attributes are all absent.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0975?>


  <xsl:template name="main">
      <out>
        <xsl:number value="3" count="a"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
