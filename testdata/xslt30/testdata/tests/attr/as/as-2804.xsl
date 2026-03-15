<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test type of global xsl:variable with @as="document-node(element(QName, derived atomic type)) ?" and @select=(). -->

   <xslt:variable name="var1"
                  select="()"
                  as="document-node(element(elem-Name,xs:Name))?"/>

   <xslt:template match="/">
      <out>
         <var1>
            <xslt:value-of select="./* instance of element(elem-Name,xs:Name)"/>
            <xslt:value-of select="$var1 instance of document-node(element(elem-Name,xs:Name))?"/>
            <xslt:value-of select="$var1 instance of document-node(element(elem-Name,xs:Name))"/>
         </var1>
      </out>
   </xslt:template>
</xslt:transform>
