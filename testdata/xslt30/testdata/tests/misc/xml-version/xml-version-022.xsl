<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
      <!-- Purpose: To output a document with method 'xml' and version '1.1'. 
                The name and namespace attributes of an xsl:attribute are evaluated
                using AVT that have a character reference (&#xD8;) for the name and 
                a character reference (&#x8C;) for the namespace, the value of the 
                separator attribute is also a character reference. -->

      <t:output method="xml" version="1.1"/>

      <t:variable name="var">preØ</t:variable>

      <t:template match="/">
            <pre:out xmlns:pre="http://example.org/preÀ"
               xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
                  <t:attribute name="{$var}" namespace="{doc/name}">
                        <t:value-of select="doc/a" separator=" *Ã* "/>
                  </t:attribute>
            </pre:out>
      </t:template>
</t:transform>
