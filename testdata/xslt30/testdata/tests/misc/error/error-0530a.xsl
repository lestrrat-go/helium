<?xml version="1.0" encoding="UTF-8"?>
<!--The value of the priority attribute  of the
                                 xsl:template element
                           must conform to the rules for the
                              xs:decimal type defined in .
                           Negative values are permitted.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                version="2.0">

<?error XTSE0530?>



  <xsl:template name="main">
      <out>
        <xsl:apply-templates/>
      </out>
  </xsl:template>
  
  <xsl:template match="doc" priority="2.0e2"/>
  
 
  


</xsl:stylesheet>
