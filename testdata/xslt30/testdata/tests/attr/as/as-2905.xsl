<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test of xsl:template with @as="schema-element($QName)". 
  				Sequence constructor contains a typed element from input file, 
  				an explicitly created xsl:element or LRE, all with the same type 
  				or a type derived from the one for $QName.-->

   <xslt:import-schema schema-location="builtinTypeSchemaAs.xsd"/>

   <xslt:template match="/">
      <out>
         <temp1>
            <xslt:call-template name="t1"/>
         </temp1>
         <temp2>
            <xslt:call-template name="t2"/>
         </temp2>
         <temp3>
            <xslt:call-template name="t3"/>
         </temp3>
      </out>
   </xslt:template>

   <xslt:template name="t1" as="schema-element(elem-anyURI)">
      <xslt:copy-of select="doc-schemaas/elem-anyURI" validation="strict"/>
   </xslt:template>

   <xslt:template name="t2" as="schema-element(elem-anyURI)">
      <elem-anyURI xslt:type="xs:anyURI">http://www.uri.com</elem-anyURI>
   </xslt:template>

   <xslt:template name="t3" as="schema-element(elem-anyURI)">
      <xslt:element name="elem-anyURI" type="derivedURI">http://test1.org</xslt:element>
   </xslt:template>
</xslt:transform>
