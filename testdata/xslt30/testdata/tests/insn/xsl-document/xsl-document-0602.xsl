<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0"
   default-validation="strip">
   <!-- Purpose: Test case that uses default-validation="strip" to test whether 
  				type annotation is stripped in an attribute constructed inside 
  				an xsl:document. -->

   <xslt:template match="doc">
      <out>
         <xslt:value-of select="$v/child[2]/@number instance of attribute(*, xs:untypedAtomic)"/>
      </out>
   </xslt:template>

   <xslt:variable name="v" as="document-node()*">
      <xslt:document>
         <child number="1">first child</child>
         <xslt:element name="child">
            <xslt:attribute name="number">2</xslt:attribute> second child</xslt:element>
      </xslt:document>
   </xslt:variable>

</xslt:transform>
