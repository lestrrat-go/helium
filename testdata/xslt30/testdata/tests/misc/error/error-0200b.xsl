<?xml version="1.0" encoding="UTF-8"?>
<!--The xsl:import element children must
                        precede all other element children of an xsl:stylesheet
                        element, including any xsl:include element children and any
                           user-defined data elements.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?error XTSE0140?>

  <my:data xmlns:my="http://my.uri/"/>
  <xsl:import href="emptymodule.xsl"/>


  <xsl:template name="main">
      <out>
         <xsl:apply-templates/>
      </out>
  </xsl:template>



</xsl:stylesheet>
