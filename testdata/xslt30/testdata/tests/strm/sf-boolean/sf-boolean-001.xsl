<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <!-- Simple use of xsl:source-document with boolean() -->
    
    <xsl:template name="c-001" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="boolean(.//ITEM)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean(), filtered with motionless predicate -->
    
    <xsl:template name="c-002" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="boolean(./BOOKLIST/BOOKS/ITEM[@CAT='P'])"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() applied to ancestor nodes-->
    
    <xsl:template name="c-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="BOOKLIST/BOOKS/ITEM[@CAT='MMP'] ! boolean(ancestor::*)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() applied to a grounded consuming sequence -->
    
    <xsl:template name="c-004" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="boolean(/BOOKLIST/BOOKS/ITEM/DIMENSIONS!tokenize(., ' '))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() applied to attributes of ancestor nodes-->
    
    <xsl:template name="c-005" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="BOOKLIST/BOOKS/ITEM[@CAT='MMP'] ! boolean(ancestor-or-self::*/@*)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() applied to namespaces of ancestor nodes-->
    
    <xsl:template name="c-006" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="BOOKLIST/BOOKS/ITEM[@CAT='MMP'] ! boolean(ancestor-or-self::*/namespace::*)"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- boolean() with empty downwards selection-->
    
    <xsl:template name="c-007" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="boolean(BOOKS)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() with empty downwards selection with predicate-->
    
    <xsl:template name="c-008" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="boolean(BOOKS/BOOK[2])"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- boolean() with a crawling (striding | striding => crawling) union -->
    
    <xsl:template name="c-009" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="boolean(BOOKLIST/BOOKS | BOOKLIST/CATEGORIES)"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- boolean() with a crawling union -->
    
    <xsl:template name="c-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="boolean(.//* | .//text())"/>
        </out>
      </xsl:source-document>
    </xsl:template>  
    
    <!-- simple motionless boolean() -->
    
    <xsl:template name="c-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:if test="child::BOOKLIST">
            <xsl:value-of select="boolean(true())"/>
          </xsl:if>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() filtered grounded sequence -->
    
    <xsl:template name="c-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="boolean(remove(data(//DIMENSIONS/text()), 3))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() filtered striding sequence -->
    
    <xsl:template name="c-013" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="boolean(remove(/BOOKLIST/BOOKS/ITEM, 3))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() twice-filtered striding sequence -->
    
    <xsl:template name="c-014" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="boolean(remove(/BOOKLIST/BOOKS/ITEM, 3)[@CAT='MMP'])"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- boolean() applied to a non-existent element -->
    
    <xsl:template name="c-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="boolean(account/transaction/details)"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- boolean() applied to an existent attribute (can exit early) -->
    
    <xsl:template name="c-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="boolean(account/transaction/@value)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with boolean() and a boolean filter -->
    
    <xsl:template name="c-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="boolean(account/transaction[@value &gt; 10000000])"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with boolean() and both a positional and a boolean filter -->
    
    <xsl:template name="c-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="boolean(account/transaction[position() lt 20][@value &gt; 1000])"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() on a sequence of both streamed nodes and atomic values -->
    
    <xsl:template name="c-100" use-when="$RUN">
      <xsl:variable name="b" select="current-date() gt xs:date('1900-01-01')"/>
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="boolean(($b, account/transaction/dummy))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() on a sequence of both streamed nodes and atomic values -->
    
    <xsl:template name="c-101" use-when="$RUN">
      <xsl:variable name="b" select="current-date() gt xs:date('1900-01-01')"/>
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="boolean((account/transaction/dummy, $b))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() on a sequence of both streamed nodes and atomic values -->
    
    <xsl:template name="c-102" use-when="$RUN">
      <xsl:variable name="b" select="current-date() gt xs:date('1900-01-01')"/>
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="boolean((account/transaction, $b))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() on a sequence of both streamed nodes and atomic values -->
    
    <xsl:template name="c-103" use-when="$RUN">
      <xsl:variable name="b" select="current-date() gt xs:date('1900-01-01')"/>
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="boolean(($b, account/transaction))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() on an xs:anyURI -->
    
    <xsl:template name="c-104" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="boolean(account/transaction[1]/base-uri(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- boolean() on an xs:date -->
    
    <xsl:template name="c-105" use-when="$RUN">
      <xsl:variable name="b" select="current-date()"/>
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="boolean((account/transaction[888888], $b))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- boolean() on an xs:date; error is caught -->
    
    <xsl:template name="c-106" use-when="$RUN">
      <xsl:variable name="b" select="current-date()"/>
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try>
            <xsl:value-of select="boolean((account/transaction[888888], $b))"/>
            <xsl:catch errors="*:FORG0006" select="'caught'"/>
          </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- test that streaming resumes OK after a caught error -->
    
    <xsl:template name="c-107" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:for-each select="account/transaction">
            <t>            
              <xsl:try>
                <xsl:value-of select="boolean(xs:double(concat('-', @value)))"/>
                <xsl:catch errors="*:FORG0001" select="'invalid'"/>
              </xsl:try>
            </t>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>
  
  <!-- boolean() applied to grounded element nodes -->
  
  <xsl:template name="c-114" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="boolean(outermost(//PRICE) ! parse-xml('&lt;p a=''3''>' || . || '&lt;/p>')/*)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- boolean() applied to grounded text nodes -->
  
  <xsl:template name="c-115" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="boolean(outermost(//PRICE) ! parse-xml('&lt;p a=''3''>' || . || '&lt;/p>')//text())"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- boolean() applied to grounded attribute nodes -->
  
  <xsl:template name="c-116" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="boolean(outermost(//PRICE) ! parse-xml('&lt;p a=''3''>' || . || '&lt;/p>')//@a)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:function name="Q{f}attribute">
    <xsl:param name="name" as="xs:string"/>
    <xsl:param name="value" as="xs:string"/>
    <xsl:attribute name="{$name}" select="$value"/>
    <!-- Atomic value after a node is OK -->
    <xsl:sequence select="25"/>
  </xsl:function>
  
  <xsl:function name="Q{f}bad-attribute">
    <xsl:param name="name" as="xs:string"/>
    <xsl:param name="value" as="xs:string"/>
    <!-- Atomic value before a node is an error -->
    <xsl:sequence select="25"/>
    <xsl:attribute name="{$name}" select="$value"/>   
  </xsl:function>
  
  <xsl:function name="Q{f}element">
    <xsl:param name="name" as="xs:string"/>
    <xsl:param name="value" as="xs:string"/>
    <xsl:element name="{$name}">
      <xsl:attribute name="x" select="'y'"/>
      <xsl:value-of select="$value"/>
    </xsl:element>
    <!-- Prevent optimisation based on type analysis -->
    <xsl:if test="current-date() lt xs:date('1900-01-01')"><xsl:sequence select="25"/></xsl:if>
  </xsl:function>
  
  <xsl:function name="Q{f}text">
    <xsl:param name="value" as="xs:string"/>
    <xsl:value-of select="$value"/>
    <!-- Prevent optimisation based on type analysis -->
    <xsl:if test="current-date() lt xs:date('1900-01-01')"><xsl:sequence select="25"/></xsl:if>
  </xsl:function>
  
  <!-- boolean() applied to constructed attribute nodes -->
  
  <xsl:template name="c-117" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="boolean(outermost(//PRICE) ! Q{f}attribute('x', string(.)))"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- boolean() applied to constructed element nodes -->
  
  <xsl:template name="c-118" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="boolean(outermost(//PRICE) ! Q{f}element('x', string(.)))"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- boolean() applied to constructed text nodes -->
  
  <xsl:template name="c-119" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="boolean(outermost(//PRICE) ! Q{f}text(string(.)))"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- boolean() applied to sequence comprising atomic value followed by attribute node -->
  
  <xsl:template name="c-120" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="boolean(outermost(//PRICE) ! Q{f}bad-attribute('x', string(.)))"/>
      </out>
    </xsl:source-document>
  </xsl:template>
    
</xsl:stylesheet>