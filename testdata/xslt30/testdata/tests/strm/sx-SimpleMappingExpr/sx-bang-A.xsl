<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://localhost/functions"
    exclude-result-prefixes="map xs f">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
   
  <!-- within xsl:source-document, use bang operator -->
   
  <xsl:template name="bang-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:sequence select="account/transaction[@value &lt; 0] ! copy-of(.)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use bang operator with numbering -->
   
  <xsl:template name="bang-002">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:sequence select="account/transaction[@value &lt; 0] ! (position(), @*!string())"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use bang operator with positional selection -->
   
  <xsl:template name="bang-003">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:sequence select="account/transaction[position() &lt; 5] ! (position(), @*!string())"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- bang operator over atomized nodes -->
   
  <xsl:template name="bang-004">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:sequence select="subsequence(data(account/transaction/@value), 5, 3) ! (position(), .)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- crawling selection, inspection body -->
   
  <xsl:template name="bang-005">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:sequence select="(//*)[position()=1 to 6] ! (position(), ':', name())"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- crawling selection, inspection body -->
   
  <xsl:template name="bang-006">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:sequence select="//* ! count(ancestor-or-self::*)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use parent axis within bang operator -->
   
  <xsl:template name="bang-008">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:sequence select="account/transaction[position() lt 5] ! name(..)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use ancestor axis within bang operator -->
   
  <xsl:template name="bang-009">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:sequence select="subsequence(account/transaction, 1, 4) ! name(ancestor::*[1])"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use subsequence() within bang operator rhs -->
   
  <xsl:template name="bang-010">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:sequence select="subsequence(account/transaction, 1, 4) ! copy-of(.)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, select attributes within bang operator rhs -->
   
  <xsl:template name="bang-011">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:sequence select="account/transaction/@value ! (if (. > 0) then string(.) else ())"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, iterate over ancestor axis within bang operator rhs -->
   
  <xsl:template name="bang-012">
    <out xmlns="http://loan.shark.com/">
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:sequence select="outermost(//*:extra) ! ancestor::* ! @* ! string()"/>
      </xsl:source-document>
    </out>
  </xsl:template>       

       
</xsl:transform>

