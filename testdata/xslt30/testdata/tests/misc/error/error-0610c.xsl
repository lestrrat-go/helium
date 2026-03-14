<?xml version="1.0" encoding="UTF-8"?>
<!--If an optional parameter has no select attribute and has an empty
                        sequence constructor, and
                     if there is an as attribute, then the default value of the
                     parameter is an empty sequence. If the empty sequence is not a valid instance
                     of the required type defined in the as attribute, then the
                     parameter is treated as a required parameter, which means that it is a non-recoverable dynamic error if the
                     caller supplies no value for the parameter.-->
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
               xmlns:xs="http://www.w3.org/2001/XMLSchema"
               version="2.0"
               exclude-result-prefixes="xs">
  <!-- Generate a data file with $records elements -->

  <xsl:param name="records" as="xs:integer"/>

  <xsl:template name="initial">
      <xsl:result-document href="{$records}.xml">
         <booklist>
            <xsl:for-each select="(1 to $records)">
               <book author="Frank Herbert030" title="Dune"/>
            </xsl:for-each>
         </booklist>
      </xsl:result-document>
  </xsl:template>
</xsl:transform>
