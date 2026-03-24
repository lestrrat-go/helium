<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">

                 <!--A non-schema-aware processor
                     must signal a static error if the stylesheet
                     includes an xsl:import-schema declaration.-->
                     
   <xsl:import-schema>
     <xs:schema>
       <xs:element name="foo"/>
     </xs:schema>
   </xsl:import-schema>
                     
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
</xsl:stylesheet>
