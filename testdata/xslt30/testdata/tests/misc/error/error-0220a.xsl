<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the synthetic
                     schema document does not satisfy the constraints described in  (section 5.1, Errors in Schema Construction and
                        Structure). This includes, without loss of generality, conflicts such
                     as multiple definitions of the same name.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?error XTSE0220?>

  <xsl:import-schema namespace="http://www.err0220.com/">
      <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
                 targetNamespace="http://www.err0220.com/">
         <xs:element name="temp"/>
         <xs:element name="temp"/>
      </xs:schema>
  </xsl:import-schema>

  <xsl:template name="main">
      <out>
         <xsl:apply-templates/>
      </out>
  </xsl:template>




</xsl:stylesheet>
