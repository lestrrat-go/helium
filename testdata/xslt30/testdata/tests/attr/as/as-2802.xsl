<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test type of global xsl:variable selecting a document node from input file, 
  				@as="document-node(schema-element(QName))". Verify that the document element 
  				is valid against the top level element declaration with name=QName.-->

   <xslt:variable name="var1" select="/" as="document-node(element(elem-Name))"/>

   <xslt:template match="/">
      <out>
         <var1>
            <xslt:value-of select="./* instance of element(elem-Name)"/>
            <xslt:value-of select="$var1 instance of document-node(element(elem-Name))"/>
         </var1>
      </out>
   </xslt:template>
</xslt:transform>
