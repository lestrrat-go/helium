<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <!-- Simple use of xsl:source-document with "treat as" -->
    
    <xsl:template name="c-001" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try><xsl:value-of select="outermost(.//node()) treat as comment()*"/><xsl:catch errors="*:XPDY0050" select="'caught'"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as", filtered with motionless predicate -->
    
    <xsl:template name="c-002" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try><xsl:copy-of select="(./BOOKLIST/BOOKS/*[@CAT='P']) treat as element(ITEM)*"/><xsl:catch errors="*" select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as" applied to ancestor nodes-->
    
    <xsl:template name="c-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try><xsl:value-of select="((BOOKLIST/BOOKS/ITEM[@CAT='MMP'] ! (ancestor::*) ) treat as element(BOOKLIST)*) ! name()"/><xsl:catch errors="*" select="'caught'"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as" applied to a grounded consuming sequence -->
    
    <xsl:template name="c-004" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM/DIMENSIONS!tokenize(., ' ')) treat as xs:string+"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as" applied to attributes of ancestor nodes-->
    
    <xsl:template name="c-005" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try><xsl:value-of select="(BOOKLIST/BOOKS/ITEM[@CAT='MMP'] ! (ancestor-or-self::*/@*))  treat as node()+"/><xsl:catch errors="*" select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as" applied to namespaces of ancestor nodes-->
    
    <xsl:template name="c-006" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try><xsl:value-of select="(BOOKLIST/BOOKS/ITEM[@CAT='MMP'] ! (ancestor-or-self::*/namespace::*))  treat as node()+"/><xsl:catch errors="*" select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- "treat as" with empty downwards selection-->
    
    <xsl:template name="c-007" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try><xsl:value-of select="(BOOKS) treat as node()+"/><xsl:catch errors="*" select="'caught'"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as" with empty downwards selection with predicate-->
    
    <xsl:template name="c-008" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try><xsl:value-of select="(BOOKS/BOOK[2]) treat as node()+"/><xsl:catch errors="*" select="'caught'"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- "treat as" with a striding union -->
    
    <xsl:template name="c-009" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try><xsl:value-of select="(BOOKLIST/*[self::BOOKS or self::CATEGORIES]) treat as element(BOOKS)+"/><xsl:catch errors="*:XPDY0050" select="'caught'"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- "treat as" with a crawling union -->
    
    <xsl:template name="c-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try><xsl:value-of select="(.//* | .//text()) treat as text()*"/><xsl:catch errors="*" select="'caught'"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>  
    
    <!-- simple motionless "treat as" -->
    
    <xsl:template name="c-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:if test="child::BOOKLIST">
            <xsl:try><xsl:value-of select="(1 to 10) treat as xs:anyAtomicType+"/><xsl:catch errors="*" select="()"/></xsl:try>
          </xsl:if>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as" filtered grounded sequence -->
    
    <xsl:template name="c-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try><xsl:value-of select="(remove(data(//DIMENSIONS/text()), 3)) treat as text()+"/><xsl:catch errors="*" select="'caught'"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as" filtered striding sequence -->
    
    <xsl:template name="c-013" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try><xsl:value-of select="(remove(/BOOKLIST/BOOKS/ITEM, 3)) treat as element()+"/><xsl:catch errors="*" select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as" twice-filtered striding sequence -->
    
    <xsl:template name="c-014" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try><xsl:value-of select="(remove(/BOOKLIST/BOOKS/ITEM, 3)[@CAT='MMP']) treat as node()?"/><xsl:catch errors="*" select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- "treat as" applied to a non-existent element -->
    
    <xsl:template name="c-015">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try><xsl:value-of select="(account/transaction/details) treat as node()+"/><xsl:catch errors="*" select="'caught'"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- "treat as" applied to an existent attribute (can exit early) -->
    
    <xsl:template name="c-016">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try><xsl:value-of select="(account/transaction/@value) treat as attribute()+"/><xsl:catch errors="*" select="()"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with "treat as" and a boolean filter -->
    
    <xsl:template name="c-017">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try><xsl:value-of select="(account/transaction[@value &gt; 10000000]) treat as node()+"/><xsl:catch errors="*" select="'caught'"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with "treat as" and both a positional and a boolean filter -->
    
    <xsl:template name="c-018">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try><xsl:value-of select="(account/transaction[position() lt 20][@value &gt; 1000]) treat as node()+"/><xsl:catch errors="*" select="'caught'"/></xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as" on a sequence of both streamed nodes and atomic values -->
    
    <xsl:template name="c-020" use-when="$RUN">
      <xsl:variable name="b" select="1[current-date() gt xs:date('1900-01-01')]"/>
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try><xsl:value-of select="(($b, account/transaction/dummy)) treat as node()+"/>
          	<xsl:catch errors="err:XPDY0050" select="'caught'"/>
          </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as" on a sequence of both streamed nodes and atomic values -->
    
    <xsl:template name="c-021" use-when="$RUN">
      <xsl:variable name="b" select="1[current-date() gt xs:date('1900-01-01')]"/>
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try><xsl:value-of select="((account/transaction/dummy, $b)) treat as node()*"/>
          	<xsl:catch errors="err:XPDY0050" select="'caught'"/>
          </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as" on a sequence of both streamed nodes and atomic values -->
    
    <xsl:template name="c-022" use-when="$RUN">
      <xsl:variable name="b" select="1[current-date() gt xs:date('1900-01-01')]"/>
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try><xsl:value-of select="((account/transaction, $b)) treat as node()*"/>
          	<xsl:catch errors="err:XPDY0050" select="'caught'"/>
          </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- "treat as" on a sequence of both streamed nodes and atomic values -->
    
    <xsl:template name="c-023" use-when="$RUN">
      <xsl:variable name="b" select="1[current-date() gt xs:date('1900-01-01')]"/>
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:try>
          	<xsl:value-of select="(($b, account/transaction)) treat as node()*"/>
          	<xsl:catch errors="err:XPDY0050" select="'caught'"/>
          </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>          
    
</xsl:stylesheet>