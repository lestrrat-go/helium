<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <!-- Purpose: Test case that uses use-when attribute on both xsl:element and its attributes
  				where the value use-when boolean value defined on the element is different 
  				from it attributes.  -->

  <t:template match="doc">
    <out>
      <t:element name="book" use-when="true()">
        <t:attribute name="chapters" select="1 to 5" use-when="false()"/>
        <t:attribute name="Title" select="'XSLT Alphabet'" use-when="true()"/>
      </t:element>
      <t:element name="Chapter" use-when="1=2">
        <t:attribute name="chapters" select="1 to 5" use-when="false()"/>
        <t:attribute name="Title" select="'XSLT Alphabet'" use-when="true()"/>
      </t:element>
    </out>
  </t:template>
</t:transform>
