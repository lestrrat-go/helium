<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:foo="http://www.foo.com"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs foo" version="2.0">
   <!-- Purpose: Test case to verify that to create the content of the new document node,
  				if there are any atomic values in the sequence, they are converted to string.-->

   <xslt:param name="p" select="1000" as="xs:double"/>

   <xslt:template match="root">
      <xslt:document>
         <out>
            <xslt:value-of select="xs:QName('foo:test')"/>
            <xslt:text>;</xslt:text>
            <xslt:value-of select="$p"/>
         </out>
      </xslt:document>
   </xslt:template>

   <xslt:template match="child"/>
</xslt:transform>
