<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xsd="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xsd" version="2.0">
   <!-- Purpose: Test type of global xsl:variable where @select contains an xs:anyURI, @as="xs:string".(type promotion)  -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="v" select="xsd:anyURI('http://www.w3.org/2001/XMLSchema')" as="xsd:string"/>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="$v instance of xsd:anyURI"/>
         <xslt:value-of select="$v instance of xsd:string"/>
      </out>
   </xslt:template>
</xslt:transform>
