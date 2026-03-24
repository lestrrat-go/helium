<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.examples.com/my"
      xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my xs"
      version="2.0">
      <!-- Purpose: Test case that creates an xsl:element and the AVT used in the name 
  				attribute delivers a result that is of type xs:QName. 
  				That will be casted to string to compute the result of the AVT.-->

      <xslt:template match="doc">
            <xslt:variable name="var" select="xs:QName('my:Var')"/>
            <out>
                  <xslt:element name="{xs:string($var)}"/>
            </out>
      </xslt:template>
</xslt:transform>
