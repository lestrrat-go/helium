<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the initial
                        template defines a template
                        parameter that specifies required="yes".-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<?error XTDE0060?>


  <xsl:template name="main">
      <xsl:param name="y" required="yes"/>
      <out>
         <x/>
      </out>
  </xsl:template>
  
</xsl:stylesheet>
