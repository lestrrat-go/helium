<xslt:transform version="2.0"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                xmlns:pre="http://pre.uri/">

<?spec xslt#xsl-element?>
<!-- BUG: Saxon 5.3.2/014 -->
<!-- DESCRIPTION: When an element is output to the result tree, and its name uses the default namespace
    prefix and a null namespace URI, but where the result tree contains an ancestor element
    in which the default namespace prefix is assigned to a non-null namespace URI, a
    namespace undeclaration (xmlns="") should be output, but isn't.  -->

<xslt:template match="pre:foo">
   <xslt:copy>
      <xslt:element name="bar"/>
   </xslt:copy>
</xslt:template>

</xslt:transform>

