<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

                    <!--It is a static error if 
                        a template rule applicable to a mode that is defined with typed="strict"
                        uses a match pattern that contains a RelativePathExprP
                        whose first StepExprP is an AxisStepP whose ForwardStepP
                        uses an axis whose principal node kind is Element and whose NodeTest
                        is an EQName that does not correspond to the name of any global element declaration
                        in the in-scope schema components.-->
                        
   <xsl:mode typed="strict"/>
   
   <xsl:import-schema>
     <xs:schema>
       <xs:element name="foo"/>
     </xs:schema>
   </xsl:import-schema>
   
   <xsl:template match="bar">
     <ouch/>
   </xsl:template>
                        
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
   
</xsl:stylesheet>
