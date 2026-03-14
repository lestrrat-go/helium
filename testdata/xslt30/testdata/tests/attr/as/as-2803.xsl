<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test of xsl:template which contains a document node from input file, 
				@as="document-node(element(QName, derived atomic type))". 
				Verify that the document element is valid.-->

   <xslt:import-schema schema-location="builtinTypeSchemaAs.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:call-template name="temp"/>
      </out>
   </xslt:template>

   <xslt:template name="temp" as="document-node(element(elem-Name,xs:Name))">
	     <xslt:copy validation="strict">
         <xslt:apply-templates/>	
	     </xslt:copy>
   </xslt:template>

   <xslt:template match="*">
      <xslt:copy>
         <xslt:copy-of select="@*, data(.)"/> 
      </xslt:copy>
   </xslt:template>
</xslt:transform>
