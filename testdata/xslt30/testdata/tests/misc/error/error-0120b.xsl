<?xml version="1.0" encoding="UTF-8"?>
<!--An xsl:stylesheet element must not have any
                     text node children.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?error XTSE0120?>


  <xsl:template name="main">
      <out>
         <xsl:apply-templates/>
      </out>
  </xsl:template>
  
  custard

</xsl:stylesheet>
