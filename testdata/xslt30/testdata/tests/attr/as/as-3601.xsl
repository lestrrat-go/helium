<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.example.com/ns/attr"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test of xsl:template with @as="schema-attribute($QName) +". 
  				Sequence constructor contains: -a typed attribute node from input file,
  				a set of typed attribute nodes from input file-->

   <xslt:import-schema namespace="http://www.example.com/ns/attr"
                       schema-location="attributesSchemaAs.xsd"/>

   <xslt:template match="/">
      <out>
         <temp1>
            <xslt:call-template name="temp1"/>
         </temp1>
         <temp2>
            <xslt:call-template name="temp2"/>
         </temp2>
      </out>
   </xslt:template>

   <xslt:template name="temp1" as="schema-attribute(my:nationality)+">
	     <xslt:copy-of select="my:people/@my:nationality" validation="strict"/>
   </xslt:template>

   <xslt:template name="temp2" as="schema-attribute(my:nationality)+">
	     <xslt:copy-of select="//@my:nationality" validation="strict"/>
   </xslt:template>
</xslt:transform>
